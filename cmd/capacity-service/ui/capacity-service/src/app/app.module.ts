import { NgModule } from '@angular/core';
import { HttpClientModule } from '@angular/common/http';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import {
  MatTableModule,
  MatCardModule,
  MatButtonModule,
  MatDialogModule,
  MatCheckboxModule,
  MatFormFieldModule,
  MatInputModule,
  MatSelectModule,
  MatSortModule,
  MatPaginatorModule,
  MatProgressSpinnerModule
} from '@angular/material'

import { AppComponent } from './app.component';
import { CapacityServiceComponent } from './capacity-service/capacity-service.component';
import { AppRoutingModule } from './/app-routing.module';
import { NodeTypesModalComponent } from './node-types-modal/node-types-modal.component';
import { ConfirmDeleteModalComponent } from './confirm-delete-modal/confirm-delete-modal.component';


@NgModule({
  declarations: [
    AppComponent,
    CapacityServiceComponent,
    NodeTypesModalComponent,
    ConfirmDeleteModalComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    FormsModule,
    ReactiveFormsModule,
    HttpClientModule,
    AppRoutingModule,
    // Material
    MatTableModule,
    MatCardModule,
    MatButtonModule,
    MatDialogModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatSortModule,
    MatPaginatorModule,
    MatProgressSpinnerModule
  ],
  providers: [],
  bootstrap: [AppComponent],
  entryComponents: [NodeTypesModalComponent,ConfirmDeleteModalComponent]
})
export class AppModule { }
