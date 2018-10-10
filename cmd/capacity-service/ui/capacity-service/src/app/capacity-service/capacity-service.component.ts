import { Component, OnInit, OnDestroy, ViewEncapsulation, ViewChild } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { MatDialog, MatTableDataSource, MatSort, MatPaginator } from '@angular/material';
import { Subscription } from 'rxjs/Subscription';
import { timer } from 'rxjs/observable/timer';
import { switchMap } from 'rxjs/operators';
import { NodeTypesModalComponent } from "../node-types-modal/node-types-modal.component";
import { ConfirmDeleteModalComponent } from "../confirm-delete-modal/confirm-delete-modal.component";

@Component({
  selector: 'app-capacity-service',
  templateUrl: './capacity-service.component.html',
  styleUrls: ['./capacity-service.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class CapacityServiceComponent implements OnInit {

  private serverEndpoint = "../api/v1";
  private configPath = this.serverEndpoint + "/config";
  private workersPath = this.serverEndpoint + "/workers";
  private machineTypesPath = this.serverEndpoint + "/machinetypes";

  public subscriptions = new Subscription();
  public config: any;
  public workers: any;
  public enableSetNodesCount: boolean;
  public currentWorkersCountMin: number;
  public currentWorkersCountMax: number;
  public nodesCountError: boolean
  public newNodeType: string;
  public allowedNodeTypes = [];
  public nodeListColumns = ["machineName", "machineType", "machineId", "reserved", "delete"];
  public nodeTypeOptions = [];
  public newNodeLoading = false;

  @ViewChild(MatSort) sort: MatSort;
  @ViewChild(MatPaginator) paginator: MatPaginator;

  constructor(private http: HttpClient, public dialog: MatDialog ) { }

  // over the wire utils
  get(path) {
    return this.http.get<any>(path)
  }

  patch(path, data) {
    return this.http.patch<any>(path, data)
  }

  post(path, data) {
    return this.http.post<any>(path, data)
  }

  delete(id) {
    return this.http.delete<any>(id)
  }


  // cs logic
  getConfig() {
    this.get(this.configPath).subscribe(
      config => {
        this.config = config;
        this.currentWorkersCountMin = this.config.workersCountMin;
        this.currentWorkersCountMax = this.config.workersCountMax;
        this.allowedNodeTypes = this.config.machineTypes;
      },
      err => console.error(err)
    );
  }

  getWorkers() {
    this.subscriptions.add(timer(0, 10000).pipe(
      switchMap(() => this.get(this.workersPath))).subscribe(
        workers => {
          // TODO: filter out terminated machines and set classes for 'pending' and 'shutting-down'
          this.workers = new MatTableDataSource(workers.items.filter(
            worker => (worker.machineState == "running" || worker.machineState == "pending")));
          this.workers.sort = this.sort;
          this.workers.paginator = this.paginator;
        },
        err => console.error(err)
      )
    )
  }

  getMachineTypes() {
    this.get(this.machineTypesPath).subscribe(
      machines => machines.forEach(m => this.nodeTypeOptions.push(m.name)),
      err => console.error(err)
    )
  }

  toggleCapService(state) {
    this.patch(this.configPath, { "paused": !state }).subscribe(
      config => this.config = config,
      err => console.error(err)
    )
  }

  editAvailableNodeTypes() {
    let modal = this.dialog.open(NodeTypesModalComponent, {
      width: "800px",
      data: { options: this.nodeTypeOptions, allowed: this.allowedNodeTypes, provider: this.config.providerName }
    })

    modal.afterClosed().subscribe(res => {
      if (res) {
        let previousAllowedNodeTypes = this.allowedNodeTypes;
        this.allowedNodeTypes = Array.from(res);
        this.patch(this.configPath, { "machineTypes": this.allowedNodeTypes }).subscribe(
          config => {
            this.config = config
          },
          err => {
            this.allowedNodeTypes = previousAllowedNodeTypes;
            console.error(err)
          }
        );
      }
    });
  }

  setNodesCount(max, min) {
    if (max >= min) {
      this.patch(this.configPath, { "workersCountMin": min, "workersCountMax": max }).subscribe(
        config => {
          this.config = config;
          this.currentWorkersCountMax = config.workersCountMax;
          this.currentWorkersCountMin = config.workersCountMin;
          // this is madness
          this.enableSetNodesCount = ((this.currentWorkersCountMax != this.config.workersCountMax) || (this.currentWorkersCountMin != this.config.workersCountMin));
        },
        err => console.error(err)
      )
    } else {
      this.updateNodesCountStatus(max, min);
    }
  }

  updateNodesCountStatus(max, min) {
    this.nodesCountError = !(max >= min);
  }

  // TODO: combine these...
  // this is madness
  incMax(e) {
    this.currentWorkersCountMax++;
    this.updateNodesCountStatus(this.currentWorkersCountMax, this.currentWorkersCountMin);
    this.enableSetNodesCount = ((this.currentWorkersCountMax != this.config.workersCountMax) || (this.currentWorkersCountMin != this.config.workersCountMin));
  }

  decMax(e) {
    if (this.currentWorkersCountMax > 0) {
      this.currentWorkersCountMax--;
      this.updateNodesCountStatus(this.currentWorkersCountMax, this.currentWorkersCountMin);
      this.enableSetNodesCount = ((this.currentWorkersCountMax != this.config.workersCountMax) || (this.currentWorkersCountMin != this.config.workersCountMin));
    }
  }

  incMin(e) {
    this.currentWorkersCountMin++;
    this.updateNodesCountStatus(this.currentWorkersCountMax, this.currentWorkersCountMin);
    this.enableSetNodesCount = ((this.currentWorkersCountMax != this.config.workersCountMax) || (this.currentWorkersCountMin != this.config.workersCountMin));
  }

  decMin(e) {
    if (this.currentWorkersCountMin > 0) {
      this.currentWorkersCountMin--;
      this.updateNodesCountStatus(this.currentWorkersCountMax, this.currentWorkersCountMin);
      this.enableSetNodesCount = ((this.currentWorkersCountMax != this.config.workersCountMax) || (this.currentWorkersCountMin != this.config.workersCountMin));
    }
  }

  toggleWorkerReserved(state, id) {
    this.patch(this.workersPath + "/" + id, { "reserved": state }).subscribe(
      res => res,
      err => console.error(err)
    )
  }

  addNewNode(type) {
    this.newNodeLoading = true;
    this.post(this.workersPath, { "machineType": type }).subscribe(
      res => {
        const data = this.workers.data;
        data.push(res);
        this.workers.data = data;
        this.newNodeType = null;
        this.newNodeLoading = false;
      },
      err => {
        console.error(err)
        this.newNodeLoading = false;
      }
    )
  }

  deleteNode(id, name) {
    const modal = this.dialog.open(ConfirmDeleteModalComponent, {
      width: "420px",
      data: { name: name }
    })

    modal.afterClosed().subscribe(res => {
      if (res) {
        this.delete(this.workersPath + "/" + id).subscribe(
          worker => {
            const data = this.workers.data;
            const updatedWorkers = data.filter(w => w.machineID != worker.machineID);
            this.workers.data = updatedWorkers;
          },
          err => console.error(err)
        )
      }
    })
  }

  ngOnInit() {
    this.getConfig();
    this.getWorkers();
    this.getMachineTypes();
    this.enableSetNodesCount = false;
  }

  ngOnDestroy() {
    this.subscriptions.unsubscribe();
  }

}
